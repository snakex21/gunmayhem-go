vx = 0;
vy = -5;
fusetime = 0;
if(asdf < 0)
{
   vx = -2 + asdf * 1.8;
   _xscale = -100;
   _rotation = _rotation - 10;
}
else
{
   vx = 2 + asdf * 1.8;
}
if(random(2) == 0)
{
   rotationspeed *= -1;
}
freepass = true;
hitground = false;
_xscale = _xscale * 0.8;
_yscale = _yscale * 0.8;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(vy < 24)
      {
         vy += _root.gravity;
      }
      if(_xscale > 0)
      {
         if(vx > 2)
         {
            vx += (2 - vx) / 10;
         }
      }
      else if(vx < -2)
      {
         vx += (-2 - vx) / 10;
      }
      _X = _X + vx;
      _Y = _Y + vy;
      if(!freepass)
      {
         if(_root.ground.hitTest(_X,_Y + 8,true) && vy > 0)
         {
            _Y = _Y - vy;
            i = 1;
            while(i <= 5)
            {
               if(_root.ground.hitTest(_X,_Y + 8 + i * (vy / 5),true))
               {
                  _Y = _Y + vy / 5 * (i - 1);
                  vy *= -0.4;
                  break;
               }
               i++;
            }
         }
      }
      if(!_root.ground.hitTest(_X,_Y,true) && !_root.ground.hitTest(_X,_Y + 10,true))
      {
         freepass = false;
      }
      if(_root.ground.hitTest(_X,_Y,true) && !freepass)
      {
         freepass = true;
      }
      fusetime += 1;
      if(fusetime > 80)
      {
         i = 0;
         while(i < _root.activeplayers.length)
         {
            distance = Math.round(Math.sqrt(Math.pow(_root.activeplayers[i]._x - _X,2) + Math.pow(_root.activeplayers[i]._y - 30 - _Y,2)));
            if(distance <= 200)
            {
               radians = Math.atan2(_root.activeplayers[i]._y - 30 - _Y,_root.activeplayers[i]._x - _X);
               degrees = radians * 180 / 3.141592653589793;
               if(_root.activeplayers[i].perknumber == 6 || _root.activeplayers[i].shieldtime > 0)
               {
                  pushx = Math.cos(degrees * 3.141592653589793 / 180) * (350 - distance) / 25;
                  pushy = Math.sin(degrees * 3.141592653589793 / 180) * (350 - distance) / 50;
                  if(_root.activeplayers[i].shield._alpha > 99)
                  {
                     _root.activeplayers[i].shield.gotoAndPlay(23);
                  }
               }
               else
               {
                  pushx = Math.cos(degrees * 3.141592653589793 / 180) * (350 - distance) / 5;
                  pushy = Math.sin(degrees * 3.141592653589793 / 180) * (350 - distance) / 10;
               }
               _root.activeplayers[i].vx += pushx;
               _root.activeplayers[i].vy += pushy;
            }
            i++;
         }
         shake = random(30) + 20;
         if(random(2) == 0)
         {
            shake *= -1;
         }
         _root._x += shake;
         _root.CP("fx_ex6",_X,_Y - 20,0);
         _root.CP("fx_ex2",_X,_Y - 20,0);
         _root.CP("fx_ex",_X,_Y - 20,0);
         _root.CP("fx_ex4",_X,_Y - 20,0);
         if(_root._quality != "LOW")
         {
            i = 0;
            while(i < 5)
            {
               _root.CP("fx_ex5",_X,_Y - 20,0);
               i++;
            }
            i = 0;
            while(i < 5)
            {
               _root.CP("fx_ex7",_X,_Y - 20,0);
               i++;
            }
         }
         _root.explodesound();
         _root.CP("fx_combo",_X,_Y - 50,0,-3);
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
      if(_Y >= 900 || _X < -500 || _X > 1400 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
