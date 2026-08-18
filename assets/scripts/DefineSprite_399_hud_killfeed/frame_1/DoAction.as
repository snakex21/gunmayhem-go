_alpha = 0;
time = 0;
if(mckiller != "none")
{
   messagebox.text = mckiller;
   messagebox3.text = mcname;
   messagebox2._x = mckiller.length * 11 + 23;
   if(_currentframe == 1)
   {
      messagebox2.text = "KILLED";
      messagebox3._x = messagebox2._x + 80;
   }
   else if(_currentframe == 3)
   {
      messagebox2.text = "CHEAPSHOT\'d";
      messagebox3._x = messagebox2._x + 140;
   }
   else if(_currentframe == 4)
   {
      messagebox2.text = "EXPLODED";
      messagebox3._x = messagebox2._x + 110;
   }
   else if(_currentframe == 5)
   {
      messagebox2.text = "GREEDYKILLED";
      messagebox3._x = messagebox2._x + 150;
   }
}
else
{
   messagebox.text = mcname;
}
stop();
this.onEnterFrame = function()
{
   time += 1;
   _Y = _Y + ((feednumber - 1) * 30 - _Y) / 3;
   if(time <= 10 && _alpha < 100)
   {
      _alpha = _alpha + 10;
   }
   if(time > 110)
   {
      _alpha = _alpha - 10;
      if(_alpha <= 1)
      {
         _parent.feednumber = _parent.feednumber - 1;
         _parent.scrollup = true;
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
   if(_parent.scrollup)
   {
      feednumber -= 1;
   }
};
