level = 1;
_rotation = random(15) + 5;
if(random(2) == 0)
{
   _rotation = _rotation * -1;
}
scaletarget = 190;
if(asdf == 1)
{
   displaytext.displaytext.text = "HIT";
}
else if(asdf > 1)
{
   displaytext.displaytext.text = asdf + " HITS";
}
else if(asdf == -1)
{
   displaytext.gotoAndStop(2);
   displaytext.displaytext.text = "SNIPED";
   scaletarget = 199.9;
}
else if(asdf == -2)
{
   displaytext.gotoAndStop(3);
   displaytext.displaytext.text = "BOOM";
   scaletarget = 199;
}
else if(asdf == -3)
{
   displaytext.gotoAndStop(3);
   displaytext.displaytext.text = "QUACK";
   scaletarget = 199;
}
else if(asdf == -4)
{
   displaytext.gotoAndStop(2);
   displaytext.displaytext.text = "PWND";
   scaletarget = 199;
}
else if(asdf == -5)
{
   displaytext.gotoAndStop(2);
   displaytext.displaytext.text = "SPLAT";
   scaletarget = 199;
}
else if(asdf == -6)
{
   displaytext.gotoAndStop(2);
   displaytext.displaytext.text = "SMACKED";
   scaletarget = 199;
}
else if(asdf == -7)
{
   displaytext.gotoAndStop(3);
   displaytext.displaytext.text = "POW";
   scaletarget = 195;
}
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(level == 1)
      {
         _xscale = _xscale + (200 - _xscale) / 3;
         _yscale = _xscale;
         if(_xscale >= scaletarget)
         {
            level = 2;
         }
      }
      if(level == 2)
      {
         _xscale = _xscale - 3;
         _yscale = _xscale;
         _alpha = _alpha - 10;
      }
      if(_alpha <= 1 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
